import http from 'k6/http';
import encoding from 'k6/encoding';
import { group } from 'k6'
import { check, sleep } from 'k6';
import { Trend } from 'k6/metrics';
import { FormData } from 'https://jslib.k6.io/formdata/0.0.2/index.js';



export const options = {
  //  executor: 'constant-vus',
   executor: 'constant-arrival-rate',
   vus: 20,
   duration: "1m",
   rps: 50,
};
const gateway = 'http://10.99.171.183:8080'
const image = open('car.jpg', 'b');
const fd = new FormData();
fd.append('payload', http.file(image, 'image.png', 'image/png'));
let resnet = {
        method: 'POST',
        // url: gateway + '/function/test-fastpod/predict',
        url: gateway + '/function/fastfunc-resnet/predict',
	// url: 'http://localhost:5000/predict',
        body: fd.body(), 
        params: {
            headers: {
	      'Content-Type': 'multipart/form-data; boundary=' + fd.boundary ,
	    },
        },
};
export default function () {
  const res = http.post(resnet.url, resnet.body, resnet.params)
  // const res = await http.asyncRequest("POST", resnet.url, resnet.body, resnet.params)
  check(res, {
    'is status 200': (r) => r.status === 200,
    //'check body': (r) => r.body.includes('car'),
  });
}