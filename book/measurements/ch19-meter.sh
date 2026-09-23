#!/usr/bin/env bash
# Task 5 metering run — steady-state sampling of the owned side.
# Samples every 20s for 10 minutes. Records:
#   - host: docker stats CPU% + memory for the cluster2 node container (whole platform)
#   - gpu:  nvidia-smi power draw (W) -- idle here, nginx uses no GPU
#   - target cluster: kubectl top nodes (control plane; worker reports <unknown>)
#   - liveness: one HTTP probe of the nginx workload per sample
OUT=meter-$(date +%Y%m%d-%H%M%S).tsv
echo -e "ts\tdocker_cpu_pct\tdocker_mem\tgpu_w\tcp_cpu_m\tcp_mem_mi\thttp_code" > "$OUT"
for i in $(seq 1 30); do
  ts=$(date -u +%H:%M:%S)
  ds=$(docker stats --no-stream --format '{{.CPUPerc}}\t{{.MemUsage}}' cluster2-control-plane 2>/dev/null)
  gw=$(nvidia-smi --query-gpu=power.draw --format=csv,noheader,nounits 2>/dev/null)
  tp=$(kubectl --kubeconfig /mnt/mil/target-cluster-kubeconfig top nodes --no-headers 2>/dev/null | grep -- '-cp-' | awk '{print $2"\t"$4}' | tr -d 'm' | sed 's/Mi//')
  hc=$(kubectl --kubeconfig /mnt/mil/target-cluster-kubeconfig get --raw "/api/v1/namespaces/default/services/nginx-meter/proxy/" >/dev/null 2>&1 && echo 200 || echo ERR)
  echo -e "${ts}\t${ds}\t${gw}\t${tp}\t${hc}" >> "$OUT"
  sleep 20
done
echo "WROTE $OUT"
wc -l "$OUT"
