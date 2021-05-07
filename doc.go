// package fastlike is a Go implementation of the Fastly Compute@Edge ABI.
//
// It is designed to be used as an `http.Handler`, and has a `ServeHTTP` method to accomodate.
// The ABI is designed around a single instance handling a single request and response pair. This
// package is thus designed accordingly, with each incoming HTTP request being passed to a fresh
// wasm instance.
// The public surface area of this package is intentionally small, as it is designed to operate on
// a single (request, response) pair and any fiddling with the internals can cause serious
// side-effects.
//
// fastly-sys ABI
// The fastly-sys ABI is the interface between a Compute wasm program and the host. In production, the
// host is Fastly's Compute platform. Fastlike is an alternative implementation of the host.
//
// The ABI is defined by the [fastly-sys](https://docs.rs/fastly-sys) crate. Note that the types
// defined in the Rust crate map to WASM types but may not always be identical. For example, in most
// cases, the Go type for an integral value is `int`. It is possible to derive the exact signature for
// a function by running `wasm2wat` (from the Web Assembly Binary Toolkit) on the generated wasm blob,
// finding the function in question, and then looking up it's type earlier in the listing.
//
// Our implementation of the ABI is in xqd*.go (prior to the `fastly-sys` crate being published, the
// ABI was referred to as the XQD API, not its simply the set of `fastly_*` modules), and it's linked
// to your wasm program via the `linker` method of the Instance type, implemented in instance.go.
//
// Each ABI method purposefully follows the signatures defined on the guest-side to make it easy to
// compare. It's not idiomatic Go by design.
//
// BACKENDS / ORIGINS
//
// In Fastly, you are expected to configure origins. These origins define where your requests will
// go once they pass through the Fastly data plane, and you cannot send requests to any origin not
// defined in your Fastly configuration. Compute programs have this same limitation. In order to
// issue a request, a Compute program must select a backend to send it to.
//
// In Fastlike, the caller is expected to provide a function which takes the name of a backend and
// an http.Request and returns an (http.Response, error) pair. The default implementation of this
// function is to return a 502 Bad Gateway.
//
// [abi.rs]: https://docs.rs/crate/fastly/0.3.2/source/src/abi.rs
// [lib.rs]: https://docs.rs/crate/fastly-shared/0.3.2/source/src/lib.rs

package fastlike
