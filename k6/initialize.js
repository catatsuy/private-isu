import { sleep } from "k6";
import http from "k6/http";
import { url } from "./config.js";

export default function () {
  http.get(url("/initialize"), {
    timeout: "10s",
  });
  sleep(1);
}
