import http from "k6/http";
import { check } from "k6";
import { parseHTML } from "k6/html";
import { url } from "./config.js";

// アップロードする画像を開く
const testImage = open("./image.png", "b");

export default function () {
  // ログイン
  const login_res = http.post(url("/login"), {
    account_name: "terra",
    password: "terraterra",
  });
  const doc = parseHTML(login_res.body);

  // csrf_token, post_id を取得
  const csrf_token = doc.find("input[name=csrf_token]").first().attr("value");
  const post_id = doc.find("input[name=post_id]").first().attr("value");

  // 画像をアップロード
  http.post(url("/"), {
    csrf_token: csrf_token,
    body: "Posted by k6",
    file: http.file(testImage, "image.png", "image/png"),
  });
}
