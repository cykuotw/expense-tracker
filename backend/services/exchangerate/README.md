# Exchange-rate recommendations

The initial recommendation source is the public
[Frankfurter v2 API](https://frankfurter.dev/). Recommendations are display
suggestions only; the application never persists provider results or applies
them without a member copying and saving a rate.

Provider review performed 2026-09-22:

- The public API requires no key and permits commercial use. It has no daily or
  monthly quota, although abuse protection is rate-limited. The client therefore
  uses one multi-currency request and a short process-local cache.
- Frankfurter is MIT-licensed. It does not state an application attribution
  requirement, but underlying source terms still apply. Recommendation responses
  include a discreet provider link for transparency.
- The live currency catalogue covered every application currency, including
  CAD, JPY, and TWD, when reviewed.
- The latest-rate response publishes a `date`, not a time. The API contract and
  UI preserve that date as `rateSnapshotAt` and must not invent a publication
  time.
- Rates are blended reference rates intended for general accounting previews,
  not live trading or payment execution.

Recheck the [service FAQ](https://frankfurter.dev/#faq), live
[currency catalogue](https://api.frankfurter.dev/v2/currencies), and underlying
provider terms before changing the registry or relying on materially different
usage volume.
