<?php

use App\Plugins\Comment\src\Model\TopicComment;

if(!function_exists("get_topic_comment_page")){
	/**
	 * 获取评论所在的页码
	 * @param int $comment_id
	 * @return int
	 */
	function get_topic_comment_page(int $comment_id): int
	{
		if(!\App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->exists()){
			return 1;
		}
		// 所在帖子ID
		$topic_id = \App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->value('topic_id');
		// 每页加载的评论数量
		$comment_num = get_options("comment_page_count",15);
		$inPage=1;
		// 获取最后一页页码
		$lastPage = TopicComment::query()
			->where(['status' => 'publish','topic_id'=>$topic_id])
			->paginate($comment_num)->lastPage();
		for($i = 0; $i < $lastPage; $i++){
			$page = $i+1;
			$data = TopicComment::query()
				->where(['status' => 'publish','topic_id'=>$topic_id])
				->with("topic","user","parent")
				->orderBy("optimal","desc")
				->paginate($comment_num,['*'],'page',$page)->items();
			foreach($data as $value){
				if((int)$value->id===(int)$comment_id){
					$inPage=$page;
				}
			}
		}
		return $inPage;
	}
}

if(!function_exists("get_topic_comment_floor")){
	/**
	 * 获取评论楼层
	 * @param int $comment_id
	 * @return int
	 */
	function get_topic_comment_floor(int $comment_id): ?int
	{
		$floor = null;
		if(!\App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->exists()){
			$floor = null;
		}
		// 所在帖子ID
		$topic_id = \App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->value('topic_id');
		// 每页加载的评论数量
		$comment_num = get_options("comment_page_count",15);
		$comment_page = get_topic_comment_page($comment_id);
		// ($key + 1)+(($comment->currentPage()-1)*get_options('comment_page_count',15))
		$page = TopicComment::query()
			->where(['topic_id' => $topic_id,'status' => 'publish'])
			->paginate($comment_num,['*'],'page',$comment_page);
		foreach($page as $k => $v){
			if((int)$v->id===$comment_id){
				$floor = ($k + 1)+(($comment_page-1)*get_options('comment_page_count',15));
			}
		}
		return $floor;
	}
}


if(!function_exists("get_topic_comment")){
	function get_topic_comment($id): array|\Hyperf\Database\Model\Collection|\Hyperf\Database\Model\Model|bool|TopicComment
	{
		$comment = TopicComment::find($id);
		if(!$comment){
			return false;
		}
		return $comment;
	}
}