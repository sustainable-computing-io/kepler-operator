## Kepler CR
The Kepler CR provides specification of each kepler components including exportor, estimator, model-server, and exporter (including prometheus and grafana dependency.
  
```yaml
collector:
    image:
    sources:
      cgroupv2: (enable|disable)
      bpf: 
      counters: 
      kubelet:
    ratio-metrics:
      global:
      core: (cpu_cycles)
      uncore: 
      dram:
estimator:
    enable: (true|false)
    image:
    strategy:
    - node-selector:
        [label]: [value]
      type: numerical
      eval-key: (mse)
      max-value:
      min-value:
    - node-selector:
        [label]: [value]
      type: list
      key: (features)
      values:
      exclude: (true|false)
    - node-selector:
        [label]: [value]
      type: string
      key: (model_name)
      value:
model-server:
    install: (true|false)
    image: 
    query-step:
    sampling-interval:
    enable-pipelines:
    - * (all available pipelines)
    - [pipeline name]
    models-storage:
      type: local
      hostPath: 
      ...
prometheus:
    # https://pkg.go.dev/github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1#PrometheusSpec
    ...
grafana:
    dashboard-script:
    ...
```