const blocked = ['adservice', 'googlesyndication', 'facbook', 'doubleclick', 'analytics', 'gemius', 'rubiconproject'];

// supported external load
// const info = require('./info.js');
// console.log(info.version)

// return false to block
function onRequest(req) {

	console.log(JSON.stringify(req))

	let host = req.Host;
	let dest = req.Header['Sec-Fetch-Dest']

	if ('video' === dest) return false;

	var allow = true;
	for (v in blocked) {
		allow = host.indexOf(blocked[v]) === -1;
		if (!allow)	break;
	}

	if (allow) {
		// console.log(host)
	} else {
		console.log('>>> BLOCK >>> ' + host)
	}

	return allow;
}
