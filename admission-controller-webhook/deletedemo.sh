#!/usr/bin/env bash
set -euo pipefail



kubectl delete -n webhook-demo service/webhook-server

kubectl delete -n webhook-demo  deployment.apps/webhook-server

kubectl delete mutatingwebhookconfigurations demo-webhook

kubectl delete ns webhook-demo