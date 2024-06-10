import json
import os

import klotho
import klotho.aws as aws
import klotho.runtime as runtime
from klotho.aws.bucket import BucketArgs
from klotho.construct import ConstructOptions

# Example usage
if __name__ == "__main__":
    # Create the Application instance
    app = klotho.Application(
        'my-app',
        project=os.getenv('PROJECT_NAME', 'my-project'),  # Default to 'my-project' or the environment variable value
        environment=os.getenv('KLOTHO_ENVIRONMENT', 'default'),
        # Default to 'default' or the environment variable value
        default_region=os.getenv('AWS_REGION', 'us-east-1'),  # Default to 'us-east-1' or the environment variable value
    )

    # Create a Container resource
    # container1 = aws.Container('my-container', 'my-image:latest')
    bucket = aws.Bucket('my-bucket', BucketArgs(
        index_document='index.html',
        sse_algorithm='AES256',
        force_destroy=True,
    ), opts=ConstructOptions())

    # b2 = aws.Bucket("my-bucket2", index_document="index.html", sse_algorithm="AES256", force_destroy=True)

#     # print(runtime.instance.generate_yaml())
# # print(str(bucket.bucket_name))
#
# all = klotho.Output.all([bucket.bucket_name, b2.bucket_name, b2.arn], lambda n1, n2, n3: f"{n1} and {n2} and {n3}")
#
# res = runtime.instance.resolve_output_references({str(bucket.urn): {
#     "BucketName": "b123",
# }, str(b2.urn): {
#     "BucketName": "b456",
# }})
#
# print([x.value for x in res])
#
#
# assert all.is_resolved is False
#
# res = runtime.instance.resolve_output_references({str(bucket.urn): {
# }, str(b2.urn): {
#     "Arn": "arn:aws:s3:::b456",
# }})
#
# print(all.value)
#
#
# print([(x.id, x.value) for x in res])

all = klotho.Output.all([bucket.bucket_name, bucket.arn], lambda n1, n2: f"{n1} and {n2}")