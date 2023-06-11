ace.define("ace/snippets/maze",["require","exports","module"],function(e,t){"use strict";t.snippetText=`snippet >
description assignment
scope maze
	-> \${1}= \${2}

snippet >
description if
scope maze
	-> IF \${2:**} THEN %\${3:L} ELSE %\${4:R}
`,t.scope="maze"}),function(){ace.require(["ace/snippets/maze"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()