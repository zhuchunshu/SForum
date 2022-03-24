<?php

namespace App\Plugins\Core\src\Lib;

use App\CodeFec\Annotation\ShortCode\ShortCodeR;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\User;

class ShortCode
{
	// 登陆可见
	#[ShortCodeR(name:"login")]
	public function login($match){
		if(@$match[1]){
			$data = $match[1];
		}else{
			$data=null;
		}
		if(auth()->check()){
			return view("Topic::ShortCode.login-show",['data'=>$data]);
		}
		
		return view("Topic::ShortCode.login-hidden",['data'=>$data]);
	}
	
	// 回复可见
	#[ShortCodeR(name:"reply")]
	public function reply($match){
		$quanxian = false;
		$topic_data = cache()->get(session()->get("view_topic_data"));
		$topic_id = $topic_data->id;
		if(auth()->check() && TopicComment::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id()])->exists()) {
			$quanxian = true;
		}
		if($quanxian === false){
			return view("Comment::ShortCode.reply-hidden",['data' => $match[1]]);
		}
		if(@$match[1]){
			$data = $match[1];
		}else{
			$data=null;
		}
		return view("Comment::ShortCode.reply-show",['data' => $data]);
	}
	
	// 密码可见
	#[ShortCodeR(name:"password")]
	public function password($match){
		if(!@$match[1] || !@$match[2]){
			return '[password]标签用法错误!';
		}
		$password = $match[1];
		$data = $match[2];
		if((string)request()->input('view-password',null)===$password){
			return view("Topic::ShortCode.password-show",['data' => $data]);
		}
		return view("Topic::ShortCode.password-hidden",['data' => $data]);
	}
	
	// 引用用户
	#[ShortCodeR(name:"user")]
	public function user($match){
		if(!@$match[1]){
			return '[user]标签用法错误!';
		}
		$user_id = $match[1];
		if(!User::query()->where('id',$user_id)->exists()){
			return $match[0];
		}
		$data = User::query()->where('id',$user_id)->first();
		return view("User::ShortCode.user",['data' => $data]);
	}
	
	// 引用评论
	#[ShortCodeR(name:"topic-comment")]
	public function topic_comment($match){
		if(!@$match[1]){
			return '[topic-comment]标签用法错误!';
		}
		$comment_id = $match[1];
		if(!TopicComment::query()->where(['id'=>$comment_id,'status' =>'publish'])->exists()){
			return $match[0];
		}
		$data = TopicComment::query()->where(['id'=>$comment_id,'status' =>'publish'])->first();
		return view("Comment::ShortCode.comment",['value' => $data]);
	}
	
	// 引用标签
	#[ShortCodeR(name:"topic-tag")]
	public function topic_tag($match){
		if(!@$match[1]){
			return '[topic-tag]标签用法错误!';
		}
		$id = $match[1];
		if(!TopicTag::query()->where(['id'=>$id])->exists()){
			return $match[0];
		}
		$data = TopicTag::query()->where(['id'=>$id])->first();
		return view("Topic::ShortCode.tag",['value' => $data]);
	}
}