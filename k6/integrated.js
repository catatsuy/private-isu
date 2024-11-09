import initialize from "./initialize.js";
import comment from "./comment.js";
import postimage from "./postimage.js";

export { initialize , comment , postimage };

export const options={
    scenarios:{
        initialize:{
            executor:"shared-iterations",
            vus:1,
            exec:'initialize',
            maxDuration:'10s'
        },
        comment:{
            executor:"constant-vus",
            vus:4,
            duration:'30s',
            exec:'comment',
            startTime:'12s',
        },
        postImage:{
            executor:"constant-vus",
            vus:2,
            duration:'30s',
            exec:"postimage",
            startTime:"12s",
        }
    }
};

export default function(){}

// k6 run integrated.js
// alp json --sort sum -r -m "/posts/[0-9]+,/@\w+" -o count,method,uri,min,avg,max,sum < ./logs/nginx/access.log