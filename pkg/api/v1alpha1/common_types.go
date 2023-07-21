package v1alpha1

const (
	ConditionReconciled string = "Reconciled"

	// ReconciledReasonComplete indicates the CR was successfully reconciled
	ReconciledReasonComplete string = "ReconcileComplete"
	//
	// ReconciledReasonError indicates an error was encountered while
	// reconciling the CR
	ReconciledReasonError string = "ReconcileError"

	InvalidKeplerObjectReason string = "InvalidKeplerResource"
)
