use fastly::experimental::uap_parse;
use fastly::geo::geo_lookup;
use fastly::http::{Method, StatusCode};
use fastly::{Body, Error, Request, Response};
use serde_json;
use std::convert::TryFrom;

const BACKEND: &str = "backend";

// The example server will send any requests for this backend to httpbin.org
const HTTPBIN: &str = "httpbin";

#[fastly::main]
fn main(mut req: Request) -> Result<Response, Error> {
    if req.get_header("httpbin-proxy").is_some() {
        return Ok(req.send(HTTPBIN)?);
    }

    match (req.get_method(), req.get_url().path()) {
        (&Method::GET, "/simple-response") => {
            Ok(Response::from_status(StatusCode::OK).with_body("Hello, world!"))
        }

        (&Method::GET, "/no-body") => {
            Ok(Response::from_status(StatusCode::NO_CONTENT).with_body(""))
        }

        (&Method::GET, "/user-agent") => {
            let result = match req.get_header(fastly::http::header::USER_AGENT) {
                Some(ua) => uap_parse(ua.to_str()?),
                None => uap_parse(""),
            };
            let s = match result {
                Ok((family, major, minor, patch)) => {
                    format!(
                        "{} {}.{}.{}",
                        family,
                        major.unwrap_or("0".to_string()),
                        minor.unwrap_or("0".to_string()),
                        patch.unwrap_or("0".to_string())
                    )
                }
                Err(_) => "error".to_string(),
            };
            Ok(Response::from_status(StatusCode::OK).with_body(s))
        }

        (&Method::GET, "/append-header") => {
            req.set_header("test-header", "test-value");
            Ok(req.send(BACKEND)?)
        }

        (&Method::GET, "/append-body") => {
            let other = Body::try_from("appended")?;
            let mut rw = Response::from_body("original\n");
            rw.append_body(other);
            rw.set_status(StatusCode::OK);
            Ok(rw)
        }

        (&Method::GET, path) if path.starts_with("/proxy") => Ok(req.send(BACKEND)?),

        (&Method::GET, "/panic!") => {
            panic!("you told me to");
        }

        (&Method::GET, "/geo") => {
            let ip = req.get_client_ip_addr();
            if ip.is_none() {
                return Ok(Response::from_status(StatusCode::INTERNAL_SERVER_ERROR).with_body(""));
            }
            let geodata = geo_lookup(ip.unwrap()).unwrap();
            Ok(Response::from_status(StatusCode::OK).with_body(
                serde_json::json!({
                    "as_name": geodata.as_name(),
                })
                .to_string(),
            ))
        }

        // This one is used for example purposes, not tests
        (&Method::GET, path) if path.starts_with("/testdata") => Ok(req.send(BACKEND)?),

        _ => Ok(Response::from_status(StatusCode::NOT_FOUND)
            .with_body("The page you requested could not be found")),
    }
}
