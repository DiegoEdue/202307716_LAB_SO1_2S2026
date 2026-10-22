import json
import os
import random
from datetime import datetime, timezone

from locust import HttpUser, between, task

# El archivo de entrada es el OCI Artifact descargado de Zot (oras pull)
CONFIG_FILE = os.getenv("CONFIG_FILE", os.path.join(os.path.dirname(__file__), "config.json"))
with open(CONFIG_FILE) as f:
    CFG = json.load(f)


class ReportUser(HttpUser):
    wait_time = between(0.05, 0.2)

    @task
    def send_report(self):
        payload = {
            "country": random.choice(CFG["countries"]),
            "warplanes_in_air": random.randint(CFG["warplanes"]["min"], CFG["warplanes"]["max"]),
            "warships_in_water": random.randint(CFG["warships"]["min"], CFG["warships"]["max"]),
            "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }
        with self.client.post(CFG["route"], json=payload, name=CFG["route"], catch_response=True) as r:
            if r.status_code != 200:
                r.failure(f"{r.status_code}: {r.text[:100]}")
