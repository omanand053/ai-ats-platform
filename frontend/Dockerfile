FROM node:20-alpine AS builder

WORKDIR /app

COPY package*.json ./
RUN npm ci

COPY . .

# 👇 Add these lines
ARG API_BACKEND_URL=http://backend:8000
ENV API_BACKEND_URL=$API_BACKEND_URL

RUN npm run build

FROM node:20-alpine

WORKDIR /app

COPY --from=builder /app/node_modules ./node_modules
COPY --from=builder /app/package.json ./package.json
COPY --from=builder /app/.next ./.next
COPY --from=builder /app/public ./public

ENV PORT=3000

EXPOSE 3000

CMD ["npm","run","start","--","-p","3000"]