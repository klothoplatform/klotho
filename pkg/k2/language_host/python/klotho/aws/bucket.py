from ..resource import Resource


class Bucket(Resource):
    def __init__(self, name):
        super().__init__(name, "klotho.aws.bucket")
