import klotho
import klotho.aws as aws
import os

# Example usage
if __name__ == "__main__":
    # Create the Application instance
    app = klotho.Application(
        'my-app',
        project=os.getenv('PROJECT_NAME', 'my-project'),  # Default to 'my-project' or the environment variable value
        environment=os.getenv('KLOTHO_ENVIRONMENT', 'default'),  # Default to 'default' or the environment variable value
        default_region=os.getenv('AWS_REGION', 'us-east-1'),  # Default to 'us-east-1' or the environment variable value
    )
    
    # Create a Container resource
    container1 = aws.Container('my-container', 'my-image:latest')
    container2 = aws.Container('my-container2', 'my-image:latest')
    container3 = aws.Container('my-container3', 'my-image:latest')
