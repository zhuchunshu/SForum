<?php

namespace App\Plugins\Blog;

/**
 * @name Blog
 * @package 开通自己的博客
 * @version 1.1.0
 * @author zhuchunshu
 * @link http://github.com/zhuchunshu
 */
class Blog
{
	public function handler(){
		$this->bootstrap();
	}
	
	public function bootstrap(){
		require __DIR__ ."/bootstrap.php";
	}
}