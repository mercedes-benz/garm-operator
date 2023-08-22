# Documentation

<!-- toc -->
- [how to](#how-to)
  - [pause reconciliation](#pause-reconciliation)
<!-- /toc -->

## how to

### pause reconciliation

In some cases, you may want to pause the reconciliation for a specific object.
This can be done by adding the annotation `garm-operator.mercedes-benz.com/paused=true` to the object.

If the annotation is present, the controller will not reconcile the object and you are able to change the object on a `garm` side without the controller reverting your changes.

#### set the annotation

```bash
$ kubectl annotate organization developer-org garm-operator.mercedes-benz.com/paused=true

organization.garm-operator.mercedes-benz.com/developer-org annotate
```

Once the annotation is set, following log message will appear in the controller log:

```
I0822 04:20:20.668789       1 organization_controller.go:74]  "msg"="Reconciliation is paused for this object" "Organization"={"name":"developer-org","namespace":"garm-operator-system"} "controller"="organization" "controllerGroup"="garm-operator.mercedes-benz.com" "controllerKind"="Organization" "name"="developer-org" "namespace"="garm-operator-system" "reconcileID"="e1737a64-b4c6-4957-b4ee-def597c215b0"
```

#### unset the annotation

To resume the reconciliation, you need to remove the annotation from the object by running:

```bash
$ kubectl annotate organization developer-org garm-operator.mercedes-benz.com/paused-

organization.garm-operator.mercedes-benz.com/developer-org annotate
```
