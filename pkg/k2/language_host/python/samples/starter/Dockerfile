# Use the official BusyBox image from the Docker Hub
FROM busybox

# Copy the HTML file to the root directory
COPY index.html /www/

# Expose port 80
EXPOSE 80

# Run httpd to serve the content of /www
CMD ["httpd", "-f", "-v", "-p", "80", "-h", "/www"]