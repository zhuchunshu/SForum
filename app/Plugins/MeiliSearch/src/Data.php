<?php

namespace App\Plugins\MeiliSearch\src;

use App\Model\AdminOption;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\User;
use Hyperf\Utils\Str;

/**
 * 数据
 */
class Data
{
	public function get(){
		$all = array_merge($this->get_topic(),$this->get_users(),$this->get_comment());
		$data = [];
		foreach($all as $key=>$value){
			$arr= [];
			$arr['id']=$key+1;
			
			$data[]=array_merge($arr,$value);
		}
		return $data;
	}
	
	// 获取全部用户
	public function get_users(){
		$arr = [];
		foreach(User::query()->get() as $data){
			$arr[]=[
				'type' => 'user',
				'title' => $data->username,
				'url' => '/users/'.$data->username.".html",
				"avatar" => $this->super_avatar($data),
				'username' => $data->username
			];
		}
		return $arr;
	}
	
	public function super_avatar($user_data): string
	{
		if($user_data->avatar){
			return $user_data->avatar;
		}
		
		if($this->get_options("core_user_def_avatar","gavatar")!=="ui-avatars") {
			return $this->get_options("theme_common_gavatar", "https://cn.gravatar.com/avatar/") . md5($user_data->email);
		}
		return "https://ui-avatars.com/api/?background=random&format=svg&name=".$user_data->username;
	}
	
	public function get_options($name,$default=""){
		return $this->core_default(@AdminOption::query()->where("name",$name)->first()->value,$default);
	}
	
	public function core_default($string=null,$default=null){
		if($string){
			return $string;
		}
		return $default;
	}
	
	// 获取全部帖子
	public function get_topic(){
		$arr = [];
		foreach(Topic::query()->get() as $data){
			$arr[]=[
				"avatar" => $this->super_avatar($data->user),
				'type' => 'topic',
				'title' => $data->title,
				'url' => '/'.$data->id.".html",
				'username' => $data->user->username
			];
		}
		return $arr;
	}
	
	// 获取全部评论
	public function get_comment(){
		$arr = [];
		foreach(TopicComment::query()->get() as $data){
			$arr[]=[
				'type' => 'comment',
				'url' => '/'.$data->topic_id.".html/".$data->id."?page=". $this->get_topic_comment_page($data->id),
				'title' => Str::limit(strip_tags($data->content),100),
				"avatar" => $this->super_avatar($data->user),
				'username' => $data->user->username
			];
		}
		return $arr;
	}
	
	public function get_topic_comment_page($comment_id): int
	{
		if(!\App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->exists()){
			return 1;
		}
		// 所在帖子ID
		$topic_id = \App\Plugins\Comment\src\Model\TopicComment::query()->where('id',$comment_id)->value('topic_id');
		// 每页加载的评论数量
		$comment_num = $this->get_options("comment_page_count",15);
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
				->orderBy("likes","desc")
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