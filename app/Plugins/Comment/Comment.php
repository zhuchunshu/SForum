<?php


namespace App\Plugins\Comment;

/**
 * Class Comment
 * @author Inkedus
 * @link https://github.com/zhuchunshu/sf-comment
 * @package 评论组件
 * @name Comment
 * @version 1.0.0
 */
class Comment
{
    public function handler(){
		$this->helpers();
        $this->setting();
    }

    public function setting(){
        require_once __DIR__."/setting.php";
    }
	
	public function helpers(){
		require_once __DIR__."/helpers.php";
	}
}