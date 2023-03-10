package controllers

import (
	"context"
	"fmt"
	"os"
	"time"

	netv1 "cloud.repo.russianpost.ru/watchdog/api/v1"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

//+kubebuilder:rbac:groups=net.post.ru,resources=watchdogs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=net.post.ru,resources=watchdogs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=net.post.ru,resources=watchdogs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=pods/exec,verbs=create
//+kubebuilder:rbac:groups="",resources=pods/status,verbs=get

const RECONCILE_INTERVAL_MIN = 2
const RECONCILE_INTERVAL_MAX = 60

// WatchdogReconciler reconciles a Watchdog object
type WatchdogReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	RESTClient rest.Interface
	RESTConfig *rest.Config
}

// Variables for metrics, more info at https://book.kubebuilder.io/reference/metrics.html
var (
	execResult = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "exec_result",
			Help: "Result of command from pod",
		},
		[]string{
			// name + namespace of CR (watchdog)
			"watchdog",
			// node IP
			"hostip",
		},
	)
)

func init() {
	metrics.Registry.MustRegister(execResult)
}

// executing command into pod with matching labels
func execIntoPod(watchdog *netv1.Watchdog, pod *corev1.Pod, r *WatchdogReconciler, logger *logr.Logger) error {
	execReq := r.RESTClient.Post().Namespace(pod.Namespace).
		Resource("pods").
		Name(pod.Name).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
			Container: "",
			Command:   watchdog.Spec.CheckCmd,
		}, runtime.NewParameterCodec(r.Scheme))

	exec, err := remotecommand.NewSPDYExecutor(r.RESTConfig, "POST", execReq.URL())

	if err != nil {
		return fmt.Errorf("error while creating remote command executor: %v", err)
	}
	logger.V(0).Info("Done", "exec=", fmt.Sprintf("%v", exec))

	return exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *WatchdogReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var watchdog netv1.Watchdog

	// pre-assign some pairs at the top of our reconcile method to have those attached to all log lines in this reconciler
	//logger := log.FromContext(ctx)
	logger := ctrl.Log.WithName("=Reconciler=")
	reconcileInterval := time.Hour

	logger.V(0).Info("In reconcile", "time", time.Now().String())
	if err := r.Get(ctx, req.NamespacedName, &watchdog); err != nil {
		logger.Error(err, "unable to fetch watchdog")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.V(0).Info("found watchdog object:", "name", watchdog.Name, "namespace", watchdog.ObjectMeta.Namespace)
	//logger.V(0).Info("labels", watchdog.Spec.ExecLabels)
	var podList corev1.PodList

	if err := r.List(ctx, &podList, client.MatchingLabels(watchdog.Spec.ExecLabels)); err != nil {
		logger.V(0).Error(err, "unable to list pod to exec into")
		return ctrl.Result{}, err
	}
	if len(podList.Items) == 0 {
		logger.V(0).Info("Pod with labels not found.")
	}
	// executing into pods and update watchdog status
	watchdog.Status.PointStatuses = nil // not sure if this assignment is needed
	watchdog.Status.PointStatuses = make([]netv1.PointStatus, 0)
	for i, item := range podList.Items {
		var metricGauge float64
		logger.V(0).Info("--->", "pod N", i, "hostIP", item.Status.HostIP)
		t := metav1.Time{Time: time.Now()}
		currentCheck := netv1.PointStatus{
			PodName:      item.ObjectMeta.Name,
			PodNamespace: item.ObjectMeta.Namespace,
			PodUID:       string(item.ObjectMeta.UID),
			HostIP:       item.Status.HostIP,
			StartTime:    &t,
			Error:        "",
		}
		if err := execIntoPod(&watchdog, &item, r, &logger); err != nil {
			currentCheck.Error = fmt.Sprintf("%v", err)
			logger.V(0).Info("error execIntoPod", "pod N", i, "exec result", currentCheck.Error)
			metricGauge = 0
		} else {
			metricGauge = 1
		}
		watchdog.Status.PointStatuses = append(watchdog.Status.PointStatuses, currentCheck)

		//update metrics
		execResult.WithLabelValues(watchdog.Name+watchdog.ObjectMeta.Namespace, item.Status.HostIP).Set(metricGauge)
	}
	if err := r.Status().Update(ctx, &watchdog); err != nil {
		logger.V(0).Error(err, "unable to update Watchdog status")
		return ctrl.Result{}, err
	}
	// TODO: can we have this optimized or static ?
	// setup next check time
	if watchdog.Spec.IntervalMinutes >= RECONCILE_INTERVAL_MIN && watchdog.Spec.IntervalMinutes <= RECONCILE_INTERVAL_MAX {
		logger.V(0).Info("correcting interval...")
		reconcileInterval = time.Duration(watchdog.Spec.IntervalMinutes) * time.Minute
	}
	logger.V(0).Info("returning", "reconcileInterval", reconcileInterval)
	return ctrl.Result{RequeueAfter: reconcileInterval}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *WatchdogReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&netv1.Watchdog{}).
		// need this predicate to avoid reconciliation of status change
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
