import http from 'k6/http';
import {check} from 'k6';
import {parseHTML} from 'k6/html';
import {url} from "./config.js";
import { getAccount } from './account.js';

const testImage = open("/Users/FujisawaNoritaka/Desktop/testImage.png","b");


export default function (){
    const account =  getAccount();
    const login_res = http.post(url("/login"),{
        account_name : account.account_name,
        password : account.password
    });

    check(login_res,{
        'is status 200':(r)=> r.status===200,
    });
    const res = http.get(url("/@terra"));

    const doc = parseHTML(res.body);
    const token = doc.find('input[name="csrf_token"]').first().attr("value");
    const comment_res = http.post(url("/"),{
        file: http.file(testImage,"testimage.jpg","image/jpeg"),
        body:"Posted by k6",
        csrf_token:token,
    });

    check(comment_res,{
        "is status 200":(r) => r.status === 200.
    });
}