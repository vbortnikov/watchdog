package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PointStatus defines check status for particular pod mathing labels
type PointStatus struct {
	// Pod's Name where the check was executed
	PodName string `json:"podName"`
	// Pod's NameSpace where the check was executed
	PodNamespace string `json:"podNamespace"`
	// Pod UID where the check was executed
	PodUID string `json:"podUID"`

	// HostIP from the status of the pod where the check was executed.
	HostIP string `json:"hostIP,omitempty"`

	// Time when check was run.
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// Error empty if no errors
	Error string `json:"error"`
}

// WatchdogSpec defines the desired state of Watchdog
type WatchdogSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Command to run in pod to check, in array form
	//+kubebuilder:validation:MaxItems=3
	//+kubebuilder:validation:MinItems=3
	CheckCmd []string `json:"checkCmd"`

	// List of labels to detect pod to run command in
	ExecLabels map[string]string `json:"execLabels"`
	// Check interval in range [5-60] minutes. Default is 1 hour.
	//+optional
	IntervalMinutes int64 `json:"intervalMinutes,omitempty"`
}

// WatchdogStatus defines the observed state of Watchdog
type WatchdogStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	PointStatuses []PointStatus `json:"pointStatuses,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Watchdog is the Schema for the watchdogs API
type Watchdog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WatchdogSpec   `json:"spec,omitempty"`
	Status WatchdogStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WatchdogList contains a list of Watchdog
type WatchdogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Watchdog `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Watchdog{}, &WatchdogList{})
}
