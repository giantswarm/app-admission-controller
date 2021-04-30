
## Emit events

At the end of each successful validation of an app CR, app-admission-controller checks the changes for the following fields and emits an event.

- `.spec.version` 
- `.spec.catalog`
- `.spec.userConfig.configMap`
- `.spec.userConfig.secret`
- `.spec.config.configMap`
- `.spec.config.secret`

The comparison occurs here [validate_app.go](../app-admission-controller/pkg/admission/validate_app.go).

If you describe the app CR with kubectl you can see the events e.g.

```
kubectl -n giantswarm describe apps userd-unique

...

Events:
  Type    Reason      Age    From                      Message
  ----    ------      ----   ----                      -------
  Normal  AppUpdated  3m2s   app-admission-controller  version has been changed to `1.1.0`
  Normal  AppUpdated  2m49s  app-admission-controller  version has been changed to `1.1.1`
  Normal  AppUpdated  1s     app-admission-controller  appConfigMap has been resetted 
```
