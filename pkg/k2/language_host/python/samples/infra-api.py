import infra
import klotho.aws as aws

api = aws.Api("my-api")
api.route_to("/hello", infra.container)
