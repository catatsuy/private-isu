import { SharedArray } from "k6/data";

const accounts = new SharedArray("accounts", function () {
  return JSON.parse(open("./accounts.json"));
});

// accounts からランダムで1件取り出す
export function getAccount() {
  return accounts[Math.floor(Math.random() * accounts.length)];
}
