// test-backends.js
import * as http from "node:http";

const ports = [3001, 3002, 3003];

ports.forEach((port) => {
  http
    .createServer((req, res) => {
      console.log(`[Backend ${port}] Received request for ${req.url}`);
      res.writeHead(200, { "Content-Type": "text/plain" });
      res.end(`Hello from Backend Server running on port ${port}!\n`);
    })
    .listen(port, () => {
      console.log(`Dummy backend listening on port ${port}`);
    });
});
