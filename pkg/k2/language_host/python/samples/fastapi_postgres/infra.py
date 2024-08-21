import os

import klotho
import klotho.aws as aws

# Create the Application instance
app = klotho.Application(
    "app",
    project=os.getenv("PROJECT_NAME", "fastapi_postgres"),
)

# Generate an absolute path to the directory containing the infra.py file /Dockerfile
dir_path = os.path.dirname(os.path.realpath(__file__))
dockerfile_path = os.path.join(dir_path, "Dockerfile")

fastapi = aws.FastAPI('my-fastapi',
                      dockerfile=dockerfile_path,
                      health_check_path="/login",
                      health_check_matcher="200-299",
                      health_check_healthy_threshold=2,
                      health_check_unhealthy_threshold=8,
                      environment_variables={
                          "PGADMIN_DEFAULT_EMAIL": "example@klo.dev",
                          "PGADMIN_DEFAULT_PASSWORD": "supsersecret123!"
                    }
                    )

postgres = aws.Postgres("my-postgres", username="admintest", password="password123!", database_name="mydb",)
fastapi.bind(postgres)

