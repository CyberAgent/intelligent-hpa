# Datadog Site

Set `DD_SITE=us5.datadoghq.com` in both the controller manager container and
all FittingJob training containers. The controller environment is not inherited
by Jobs: set `env` in the FittingJob / IHPA job patch configuration as well.

Unset or empty `DD_SITE` preserves the existing US1 endpoint. Supported sites:
`datadoghq.com`, `us3.datadoghq.com`, `us5.datadoghq.com`, `datadoghq.eu`,
`ap1.datadoghq.com`, `ap2.datadoghq.com`. Unknown sites fail before HTTP requests.

The controller applies this setting to metric queries, forecast submission and
metric metadata calls. FittingJob applies it to historical metric queries.
Rebuild both controller and FittingJob images before configuring the environment.
This does not change models, CRD schemas, existing dashboards or Datadog keys.
