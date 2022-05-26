<?php

namespace App\Plugins\Topic\src\Handler\Topic;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\Topic\src\Models\TopicUpdated;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UserClass;
use Hyperf\Utils\Str;
use Psr\SimpleCache\InvalidArgumentException;

class EditTopic
{
	public function handler($request)
	{
		if($this->validate($request) !== true) {
			return $this->validate($request);
		}
		$this->update($request);
		return Json_Api(200, true, ['修改成功!', '2秒后跳转到当前帖子页面']);
	}
	
	
	public function update($request): void
	{
		// 帖子id
		$topic_id = $request->input("topic_id");
		// 帖子标题
		$title = $request->input("title");
		// 帖子标签
		$tag = $request->input("tag");
		// 帖子md内容
		$markdown = $request->input("markdown");
		// 帖子html内容
		$html = $request->input("html");
		$options = [];
		// xss过滤
		$html = xss()->clean($html);
		
		// 解析标签
		$yhtml = $html;
		$html = $this->tag($html);
		// 解析艾特
		$html = $this->at($html);
		
		$post_id = Topic::query()->find($topic_id)->post_id;
		Post::query()->where('id',$post_id)->update([
			"content" => $html,
			"markdown" => $markdown,
		]);
		Topic::query()->where("id", $topic_id)->update([
			"title" => $title,
			"tag_id" => $tag,
		]);
		$data = Topic::query()->where("id", $topic_id)->first();
		TopicUpdated::create([
			"topic_id" => $topic_id,
			"user_id" => auth()->id(),
			'user_ip' => get_client_ip(),
			'user_agent' => get_user_agent(),
		]);
		$this->topic_keywords($data, $yhtml);
		$topic_data = Topic::query()->where("id", $topic_id)->first();
		$this->at_user($topic_data, $yhtml);
		cache()->delete("topic.data." . $topic_id);
	}
	
	private function at_user(\Hyperf\Database\Model\Model|\Hyperf\Database\Model\Builder $data, string $html): void
	{
		$at_user = get_all_at($html);
		foreach($at_user as $value) {
			go(function() use ($value, $data) {
				if(User::query()->where("username", $value)->exists()) {
					$user = User::query()->where("username", $value)->first();
					if($user->id != $data->user_id) {
						user_notice()->send($user->id, "有人在帖子中提到了你", $user->username . "在帖子<b>" . $data->title . "</b>中提到了你", "/" . $data->id . ".html");
					}
				}
			});
		}
	}
	
	
	public function tag(string $html)
	{
		$html = replace_all_keywords($html);
		return $html;
	}
	
	public function at(string $html): string
	{
		return replace_all_at($html);
	}
	
	public function topic_keywords($data, string $html): void
	{
		foreach(get_all_keywords($html) as $tag) {
			if(!TopicKeyword::query()->where("name", $tag)->exists()) {
				TopicKeyword::query()->create([
					"name" => $tag,
					"user_id" => auth()->id()
				]);
			}
			$tk = TopicKeyword::query()->where("name", $tag)->first();
			if(!TopicKeywordsWith::query()->where(['topic_id' => $data->id, 'with_id' => $tk->id])->exists()) {
				TopicKeywordsWith::query()->create([
					'topic_id' => $data->id,
					'with_id' => $tk->id,
					'user_id' => auth()->id()
				]);
			}
		}
	}
	
	private function validate($request): array|bool
	{
		$class_name = UserClass::query()->where('id', auth()->data()->class_id)->first()->name;
		$tag_value = TopicTag::query()->where("id", $request->input('tag'))->first();
		if(!user_TopicTagQuanxianCheck($tag_value, $class_name)) {
			return Json_Api(401, false, ['无权使用此标签']);
		}
		return true;
	}
}