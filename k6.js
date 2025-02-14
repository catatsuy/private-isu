import http from "k6/http";

const url = "http://localhost";

export default function () {
  http.get(url);
}
