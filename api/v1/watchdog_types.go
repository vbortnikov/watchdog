package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
