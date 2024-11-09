import http from "k6/http";

const BASE_URL = "http://localhost";

export default function(){
    http.get(`${BASE_URL}/`);
}