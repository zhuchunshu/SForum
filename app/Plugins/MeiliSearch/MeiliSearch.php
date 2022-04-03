<?php

namespace App\Plugins\MeiliSearch;

class MeiliSearch
{
	public function handler(){
		$this->bootstrap();
	}
	
	public function bootstrap(){
		require_once __DIR__ ."/bootstrap.php";
	}
}