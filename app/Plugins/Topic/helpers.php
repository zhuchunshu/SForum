<?php
// 获取帖子信息
use App\Plugins\Topic\src\ContentParse;
use App\Plugins\Topic\src\Models\Topic;

if(!function_exists("get_topic")){
	function get_topic($id): array|\Hyperf\Database\Model\Collection|\Hyperf\Database\Model\Model|bool|Topic
	{
		$topic = Topic::find($id);
		if(!$topic){
			return false;
		}
		return $topic;
	}
}

if(!function_exists("ContentParse")){
	function ContentParse(): ContentParse
	{
		return new ContentParse();
	}
}
