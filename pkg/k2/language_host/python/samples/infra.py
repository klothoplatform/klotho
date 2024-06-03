from sdk.klotho import Container, Application
import os

# Example usage
if __name__ == "__main__":
    # Create the Application instance
    app = Application(
        'my-app',
        project=os.getenv('PROJECT_NAME', 'my-project'),  # Default to 'my-project' or the environment variable value
        environment=os.getenv('KLOTHO_ENVIRONMENT', 'default'),  # Default to 'default' or the environment variable value
        default_region=os.getenv('AWS_REGION', 'us-east-1'),  # Default to 'us-east-1' or the environment variable value
    )
    
    # Create a Container resource
    container1 = Container('my-container', 'my-image:latest')
  