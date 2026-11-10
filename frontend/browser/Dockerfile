FROM mcr.microsoft.com/playwright:v1.64.1-noble@sha256:47b510dd9230bf0b9bfc9b20102241e577ea7a99334ab4c3ca19f19770e9b8ea
ARG NODE_VERSION
ARG NODE_ARCH
ARG NODE_ARCHIVE_SHA256
RUN curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/node-v${NODE_VERSION}-linux-${NODE_ARCH}.tar.gz" -o /tmp/node.tar.gz \
    && echo "${NODE_ARCHIVE_SHA256}  /tmp/node.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/node.tar.gz -C /usr/local --strip-components=1 \
    && rm /tmp/node.tar.gz
COPY pnpm-bootstrap/package.json pnpm-bootstrap/package-lock.json /opt/pnpm/
RUN npm --prefix /opt/pnpm ci --ignore-scripts --no-audit --no-fund \
    && node /opt/pnpm/node_modules/pnpm/install.js
ENV PATH="/opt/pnpm/node_modules/.bin:${PATH}"
WORKDIR /opt/dependencies
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts \
    && chmod -R a+rX /opt/dependencies
USER pwuser
