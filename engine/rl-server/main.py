import grpc
from concurrent import futures
import time

from proto.gen.engine import engine_pb2_grpc, engine_pb2


class FuturaOptimizerServicer(engine_pb2_grpc.FuturaOptimizerServicer):
    def SyncClusterOptimizationConfig(self, request, context):
        print("Received request:", request)
        return engine_pb2.ClusterOptimizationConfigResponse(success=True, message="hello world")

    def SyncServiceLevelObjective(self, request, context):
        print("Received request:", request)
        return engine_pb2.SyncSLOResponse(success=True, message="hello world")

    def GetOptimizationDecision(self, request, context):
        print("Received request:", request)
        return engine_pb2.DecisionResponse(actions=[], decision_id="",
                                           expected_outcomes=engine_pb2.ExpectedOutcomes(
                                               predicted_latency_p95_ms=0, predicted_cost_delta_per_hour_usd=0),
                                           target=engine_pb2.TargetRef(kind="", name="", namespace=""))


def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    engine_pb2_grpc.add_FuturaOptimizerServicer_to_server(
        FuturaOptimizerServicer(), server)

    server.add_insecure_port("[::]:50055")
    server.start()
    print("gRPC server running on port 50055...")
    try:
        while True:
            time.sleep(60*60*24)  # Keep alive
    except KeyboardInterrupt:
        server.stop(0)


if __name__ == "__main__":
    serve()
