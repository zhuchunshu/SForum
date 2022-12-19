<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Handler\Topic\Middleware\Create;

use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Handler\Topic\Middleware\MiddlewareInterface;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\Topic\src\Models\TopicKeyword;
use App\Plugins\Topic\src\Models\TopicKeywordsWith;
use App\Plugins\User\src\Models\User;
use Hyperf\Di\Annotation\Inject;
use Hyperf\Validation\Contract\ValidatorFactoryInterface;

#[\App\Plugins\Topic\src\Annotation\Topic\CreateMiddleware]
class CreateMiddleware implements MiddlewareInterface
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
                'content' => 'required|string|min:' . get_options('topic_create_content_min', 10),
                'title' => 'required|string|min:' . get_options('topic_create_title_min', 1) . '|max:' . get_options('topic_create_title_max', 200),
                'tag' => 'required|exists:topic_tag,id',
            ],
            [
                'content' => '内容',
                'title' => '标题',
                'tag' => '标签',
            ]
        );

        if ($validator->fails()) {
            // Handle exception
            return redirect()->with('danger', $validator->errors()->first())->url('topic/create')->go();
        }
        $this->create($data);
        return $next($data);
    }

    public function create($data): void
    {
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

        $post = Post::query()->create([
            'content' => $content,
            'markdown' => '',
            'user_id' => auth()->id(),
            'user_ip' => get_client_ip(),
            'user_agent' => get_user_agent(),
        ]);
        $data = Topic::query()->create([
            'post_id' => $post->id,
            'title' => $title,
            'user_id' => auth()->id(),
            'status' => 'publish',
            'view' => 0,
            'tag_id' => $tag,
        ]);
        // 给Posts表设置topic_id字段的值
        Post::query()->where('id', $post->id)->update(['topic_id' => $data->id]);
        $this->topic_keywords($data, $_content);
        $this->at_user($data, $_content);
    }

    private function tag(string $html)
    {
        return replace_all_keywords($html);
    }

    private function at(string $html): string
    {
        return replace_all_at($html);
    }

    private function topic_keywords($data, string $html): void
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
