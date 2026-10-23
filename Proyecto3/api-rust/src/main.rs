use axum::{extract::State, http::StatusCode, routing::{get, post}, Json, Router};
use serde::{Deserialize, Serialize};
use std::{env, sync::Arc, time::Duration};

#[derive(Deserialize, Serialize)]
struct Report {
    country: String,
    warplanes_in_air: i32,
    warships_in_water: i32,
    timestamp: String,
}

struct AppState {
    client: reqwest::Client,
    target: String,
    log_requests: bool,
}

async fn health() -> &'static str {
    "OK"
}

// Reenvia el reporte al Deployment 1 de Go y devuelve su respuesta tal cual
async fn forward(State(st): State<Arc<AppState>>, Json(rep): Json<Report>) -> (StatusCode, String) {
    match st.client.post(&st.target).json(&rep).send().await {
        Ok(resp) => {
            let status = StatusCode::from_u16(resp.status().as_u16())
                .unwrap_or(StatusCode::BAD_GATEWAY);
            let body = resp.text().await.unwrap_or_default();
            if st.log_requests {
                println!(
                    "reporte recibido: pais={} aviones={} barcos={} -> go-api respondio {}",
                    rep.country, rep.warplanes_in_air, rep.warships_in_water, status.as_u16()
                );
            }
            (status, body)
        }
        Err(e) => {
            eprintln!("error contactando go-api: {e}");
            (StatusCode::BAD_GATEWAY, "go-api no disponible".to_string())
        }
    }
}

#[tokio::main]
async fn main() {
    let carnet = env::var("CARNET").unwrap_or_else(|_| "202307716".to_string());
    let target = env::var("GO_API_URL")
        .unwrap_or_else(|_| "http://go-api-svc:8080/report".to_string());

    let client = reqwest::Client::builder()
        .timeout(Duration::from_secs(10))
        .build()
        .unwrap();
    let log_requests = env::var("LOG_REQUESTS").map(|v| v != "false").unwrap_or(true);
    let state = Arc::new(AppState { client, target, log_requests });

    let app = Router::new()
        .route("/", get(health))
        .route(&format!("/grpc-{carnet}"), post(forward))
        .with_state(state);

    let listener = tokio::net::TcpListener::bind("0.0.0.0:8080").await.unwrap();
    println!("api-rust escuchando en :8080 (ruta /grpc-{carnet})");
    axum::serve(listener, app).await.unwrap();
}