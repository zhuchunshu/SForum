ace.define("ace/snippets/csound_document",["require","exports","module"],function(e,t){"use strict";t.snippetText=`# <CsoundSynthesizer>
snippet synth
	<CsoundSynthesizer>
	<CsInstruments>
	\${1}
	</CsInstruments>
	<CsScore>
	e
	</CsScore>
	</CsoundSynthesizer>
`,t.scope="csound_document"}),function(){ace.require(["ace/snippets/csound_document"],function(e){typeof module=="object"&&typeof exports=="object"&&module&&(module.exports=e)})}()