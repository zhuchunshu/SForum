ace.define("ace/snippets/snippets",["require","exports","module"],function(e,t){"use strict";t.snippetText=`# snippets for making snippets :)
snippet snip
	snippet \${1:trigger}
		\${2}
snippet msnip
	snippet \${1:trigger} \${2:description}
		\${3}
snippet v
	{VISUAL}
`,t.scope="snippets"}),function(){ace.require(["ace/snippets/snippets"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()