# Production stage
FROM nginx:alpine

# Copy built assets
COPY dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
