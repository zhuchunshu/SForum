<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Update;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use App\Plugins\Topic\src\Models\TopicUpdated;
use App\Plugins\User\src\Models\User;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;

#[\App\Plugins\Topic\src\Annotation\Topic\UpdateMiddleware]
class UpdateMiddleware implements MiddlewareInterface
{
    /**
     * @Inject
     */
    protected ValidatorFactoryInterface $validationFactory;

    public function handler($data, \Closure $next)
    {
        $validator = $this->validationFactory->make(
            $data['basis'],
            [
                'topic_id' => 'required|exists:topic,id',
                'content' => 'required|string|min:' . get_options('topic_create_content_min', 10),
                'title' => 'required|string|min:' . get_options('topic_create_title_min', 1) . '|max:' . get_options('topic_create_title_max', 200),
                'tag' => 'required|exists:topic_tag,id',
            ],
            [],
            [
                'topic_id' => '帖子id',
                'content' => '正文内容',
                'title' => '标题',
                'tag' => '标签id',
            ]
        );

        if ($validator->fails()) {
            return redirect()->with('danger', $validator->errors()->first())->back()->go();
        }
        $data['topic_id'] = $data['basis']['topic_id'];
        $topic = Topic::query()->find($data['topic_id']);
        $quanxian = false;
        if(Authority()->check("admin_topic_edit") && auth()->Class()['permission-value']>curd()->GetUserClass($topic->user->class_id)['permission-value']){
            $quanxian = true;
        }
        if(Authority()->check("topic_edit") && auth()->id() === $topic->user->id){
            $quanxian = true;
        }
        if($quanxian===false){
            return redirect()->back()->with('danger','无权修改!')->go();
        }
        $data = $this->update($data);
        return $next($data);
    }

    public function update($data)
    {
        // 帖子id
        $topic_id = $data['topic_id'];
        // 帖子标题
        $title = $data['basis']['title'];
        // 帖子标签
        $tag = $data['basis']['tag'];
        // 帖子html内容
        $content = $data['basis']['content'];
        $content = xss()->clean($content);

        // 解析标签
        $_content = $content;
        $content = $this->tag($content);
        // 解析艾特
        $content = $this->at($content);

        $post_id = Topic::query()->find($topic_id)->post_id;
        Post::query()->where('id', $post_id)->update([
            'content' => $content,
        ]);
        Topic::query()->where('id', $topic_id)->update([
            'title' => $title,
            'tag_id' => $tag,
        ]);
        $topic = Topic::query()->where('id', $topic_id)->first();
        TopicUpdated::create([
            'topic_id' => $topic_id,
            'user_id' => auth()->id(),
            'user_ip' => get_client_ip(),
            'user_agent' => get_user_agent(),
        ]);
        $this->topic_keywords($topic, $_content);
        $topic_data = Topic::query()->find($topic_id);
        $this->at_user($topic_data, $_content);
        cache()->delete('topic.data.' . $topic_id);
        $data['topic_id'] = $topic_id;
        $data['post_id'] = $post_id;
        return $data;
    }

    public function tag(string $html)
    {
        return replace_all_keywords($html);
    }

    public function at(string $html): string
    {
        return replace_all_at($html);
    }

    public function topic_keywords($data, string $html): void
    {
        foreach (get_all_keywords($html) as $tag) {
            if (! TopicKeyword::query()->where('name', $tag)->exists()) {
                TopicKeyword::query()->create([
                    'name' => $tag,
                    'user_id' => auth()->id(),
                ]);
            }
            $tk = TopicKeyword::query()->where('name', $tag)->first();
            if (! TopicKeywordsWith::query()->where(['topic_id' => $data->id, 'with_id' => $tk->id])->exists()) {
                TopicKeywordsWith::query()->create([
                    'topic_id' => $data->id,
                    'with_id' => $tk->id,
                    'user_id' => auth()->id(),
                ]);
            }
        }
    }

    private function at_user(\Hyperf\Database\Model\Model | \Hyperf\Database\Model\Builder $data, string $html): void
    {
        $at_user = get_all_at($html);
        foreach ($at_user as $value) {
            go(function () use ($value, $data) {
                if (User::query()->where('username', $value)->exists()) {
                    $user = User::query()->where('username', $value)->first();
                    if ($user->id != $data->user_id) {
                        user_notice()->send($user->id, '有人在帖子中提到了你', $user->username . '在帖子<b>' . $data->title . '</b>中提到了你', '/' . $data->id . '.html');
                    }
                }
            });
        }
    }
}
