//! Default Compute@Edge template program.

use fastly::http::{HeaderValue, Method, StatusCode};
//use fastly::request::CacheOverride;
use fastly::{Body, Error, Request, RequestExt, Response, ResponseExt};
use fastly::downstream_request;
use std::convert::TryFrom;

/// The name of a backend server associated with this service.
///
/// This should be changed to match the name of your own backend. See the the `Hosts` section of
/// the Fastly WASM service UI for more information.
const BACKEND_NAME: &str = "backend_name";

/// The name of a second backend associated with this service.
const OTHER_BACKEND_NAME: &str = "other_backend_name";

// A do-nothing entrypoint...maybe we can execute this.
#[no_mangle]
fn main2() -> () {
    match inner() {
        Ok(_) => (),
        Err(e) => {
            println!("Unexpected error: {}", e);
            ()
        }
    }
}

fn inner() -> Result<(), Error> {
    println!("getting request");
    let r = downstream_request()?;

    println!("got request");
    println!("request method={}; uri={}", r.method().as_str(), r.uri());
    println!("building response");
    let w = Response::builder()
        .status(StatusCode::METHOD_NOT_ALLOWED)
        .body(Body::try_from("This method is not allowed")?)?;
    println!("returning..");
    w.send_downstream()
}

/// The entrypoint for your application.
///
/// This function is triggered when your service receives a client request. It could be used to
/// route based on the request properties (such as method or path), send the request to a backend,
/// make completely new requests, and/or generate synthetic responses.
///
/// If `main` returns an error a 500 error response will be delivered to the client.
#[fastly::main]
fn main(mut req: Request<Body>) -> Result<impl ResponseExt, Error> {
    // Make any desired changes to the client request
    req.headers_mut()
        .insert("Host", HeaderValue::from_static("example.com"));

    // We can filter requests that have unexpected methods.
    const VALID_METHODS: [Method; 3] = [Method::HEAD, Method::GET, Method::POST];
    if !(VALID_METHODS.contains(req.method())) {
        return Ok(Response::builder()
            .status(StatusCode::METHOD_NOT_ALLOWED)
            .body(Body::try_from("This method is not allowed")?)?);
    }

    // Pattern match on the request method and path.
    match (req.method(), req.uri().path()) {
        // If request is a `GET` to the `/` path, send a default response.
        (&Method::GET, "/") => Ok(Response::builder()
            .status(StatusCode::OK)
            .body(Body::try_from("Welcome to Fastly Compute@Edge!")?)?),

        // If request is a `GET` to the `/backend` path, send to a named backend.
        (&Method::GET, "/backend") => {
            // Request handling logic could go here...
            // Eg. send the request to an origin backend and then cache the
            // response for one minute.
            //req.set_cache_override(CacheOverride::ttl(60));
            req.send(BACKEND_NAME)
        }

        // If request is a `GET` to a path starting with `/other/`.
        (&Method::GET, path) if path.starts_with("/other/") => {
            // Send request to a different backend and don't cache response.
            //req.set_cache_override(CacheOverride::Pass);
            req.send(OTHER_BACKEND_NAME)
        }

        // Catch all other requests and return a 404.
        _ => Ok(Response::builder()
            .status(StatusCode::NOT_FOUND)
            .body(Body::try_from("The page you requested could not be found")?)?),
    }
}
