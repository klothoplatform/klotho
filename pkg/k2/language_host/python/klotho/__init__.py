from .klotho import KlothoSDK
from .application import Application
from .resource import Resource

# Function to get the singleton instance
def get_klotho():
    return KlothoSDK()
