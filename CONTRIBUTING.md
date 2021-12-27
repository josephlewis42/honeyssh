## Contributing

**Guiding rules**

* The honeypot should be interesting enough to capture payloads, but not
  interesting or interactive enough to allow further attack.
* We are not smarter than a dedicated adversary, the honeypot should be
  easy to sandbox and easy to read/understand.
* The honeypot is a honeypot. Each additional feature and library make it harder
  to maintain, be judicious.

Contributions are welcome under the following circumstances:

1. It fixes a clear bug or security issue with the honeypot.
1. It's a feature that allows more payloads to be captured.
1. They're tested in a way similar to the surrounding code.

If you are uncertain about a feature or want to make a larger change, please
open an issue first to discuss it.

Welcome changes that probably need some design are:

* Replacing Afero with a better FS library that doesn't have so many bugs.
* Using real commands running in WASM sandboxes.
* Generating FS tests from https://github.com/zfsonlinux/fstest
