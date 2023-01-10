<?php

namespace App\Plugins\Docs\src;

use App\Plugins\Docs\src\Model\Docs;

class Seo
{
	public function get_class(){
		$urls = [];
		foreach(Docs::query()->get() as $value){
			$urls[]=url("/docs/".$value->id);
		}
		return $urls;
	}
}