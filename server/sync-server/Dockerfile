FROM node:22-alpine
WORKDIR /app
COPY package.json tsconfig.json ./
COPY src ./src
COPY migrations ./migrations
RUN npm install -g pnpm && pnpm install --frozen-lockfile=false
RUN pnpm build
EXPOSE 8140
CMD ["pnpm", "start"]
