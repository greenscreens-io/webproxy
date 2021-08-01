/*
console.log(bodyContent.length)
console.log(typeof bodyContent)
console.log(bodyContent.replace)
console.log(bodyContent.replaceAll)
*/
function onBodyResponse(bodyContent) {
	if (bodyContent == null) return null;
	var css = 'video,script,iframe,.intextAdIgnore,.css-oubpvp';
	var script = "<script>setInterval(() => {document.querySelectorAll('"+css+"').forEach( el => el.remove());}, 1000);</script></body>";
	return bodyContent.replace(/<video/g, '<v').replace(/video>/g, 'v>').replace('</body>', script);
}
