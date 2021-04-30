
## Emit events

At the end of successful validation of app CRs, app-admission-controllers check the changes from the following specs and emit each of events.
- `.sepc.version` 
- `.sepc.catalog`
- `.sepc.userConfig.configMap`
- `.sepc.userConfig.secret`
- `.sepc.config.configMap`
- `.sepc.config.secret`

To see where comparison occurs, see [validate_app.go](../app-admission-controller/pkg/admission/validate_app.go).


If you describe the app CR objects in kubectl, you can see the `AppUpdated` events as below.

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
