## Log output app

Deploy with `kubectl apply -f ping_pong/manifests
kubectl apply -f log_output/manifests
kubectl apply -f persistenvolumes
curl -X GET localhost:8081/pingpong
curl localhost:8081/`
