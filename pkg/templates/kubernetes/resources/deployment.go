qualified_type_name: kubernetes:deployment
display_name: Deployment

properties:
  Cluster:
    type: resource
    operational_rule:
      steps:
        - direction: downstream
          resources:
            - classifications:
                - cluster
                - kubernetes
  Object:
    type: map
    properties:
      Kind:
        type: string
        default_value: Deployment

delete_context:
  requires_no_upstream: true
views:
  dataflow: big