<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Lib;

use App\CodeFec\Annotation\ShortCode\ShortCodeR;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\InvitationCode;
use App\Plugins\Topic\src\Models\TopicTag;
use App\Plugins\User\src\Models\User;
use Hyperf\Stringable\Str;
use Thunder\Shortcode\Shortcode\ShortcodeInterface;
class ShortCode
{
    // 登陆可见
    #[ShortCodeR(name: 'login')]
    public function login($match)
    {
        if (@$match[1]) {
            $data = $match[1];
        } else {
            $data = null;
        }
        if (auth()->check()) {
            return view('Topic::ShortCode.login-show', ['data' => $data]);
        }
        return view('Topic::ShortCode.login-hidden', ['data' => $data]);
    }
    // 回复可见
    #[ShortCodeR(name: 'reply')]
    public function reply($match, ShortcodeInterface $s, $data)
    {
        $quanxian = false;
        $topic_data = $data['topic'];
        $topic_id = $topic_data->id;
        if (auth()->check() && TopicComment::query()->where(['topic_id' => $topic_id, 'user_id' => auth()->id()])->exists()) {
            $quanxian = true;
        }
        if (auth()->check() && (int) $topic_data->user_id === auth()->id()) {
            $quanxian = true;
        }
        if ($quanxian === false) {
            return view('Comment::ShortCode.reply-hidden', ['data' => $match[1]]);
        }
        if (@$match[1]) {
            $data = $match[1];
        } else {
            $data = null;
        }
        return view('Comment::ShortCode.reply-show', ['data' => $data]);
    }
    // 密码可见
    #[ShortCodeR(name: 'password')]
    public function password($match, ShortcodeInterface $s, $d)
    {
        $s->getContent();
        if (!@$match[1] || !@$match[2]) {
            return '[password]标签用法错误!';
        }
        $password = $s->getParameter('password');
        $data = $s->getContent();
        $topic_data = $d['topic'];
        if ((string) request()->input('view-password', null) === $password || @(int) $topic_data->user_id === auth()->id()) {
            return view('Topic::ShortCode.password-show', ['data' => $data]);
        }
        return view('Topic::ShortCode.password-hidden', ['data' => $data]);
    }
    // 引用用户
    #[ShortCodeR(name: 'user')]
    public function user($match, ShortcodeInterface $s)
    {
        $user_id = $s->getParameter('user_id');
        if (!User::query()->where('id', $user_id)->orWhere('username', $user_id)->exists()) {
            return '[' . $s->getName() . '] ' . __('app.Error using short tags');
        }
        $data = User::query()->where('id', $user_id)->orWhere('username', $user_id)->first();
        return view('User::ShortCode.user', ['data' => $data]);
    }
    // 引用评论
    #[ShortCodeR(name: 'comment')]
    public function topic_comment($match, ShortcodeInterface $s)
    {
        $comment_id = $s->getParameter('comment_id');
        if (!TopicComment::where(['id' => $comment_id])->exists()) {
            return '[' . $s->getName() . '] ' . __('app.Error using short tags');
        }
        $data = TopicComment::find($comment_id);
        return view('Comment::ShortCode.comment', ['value' => $data]);
    }
    // 引用板块
    #[ShortCodeR(name: 'topic-tag')]
    public function topic_tag($match, ShortcodeInterface $s)
    {
        $id = $s->getParameter('tag_id');
        if (!TopicTag::query()->where(['id' => $id])->exists()) {
            return '[' . $s->getName() . '] ' . __('app.Error using short tags');
        }
        $data = TopicTag::query()->where(['id' => $id])->first();
        return view('Topic::ShortCode.tag', ['value' => $data]);
    }
    #[ShortCodeR(name: 'InvitationCode')]
    public function InvitationCode($match, ShortcodeInterface $s, $data)
    {
        $user_id = 0;
        if (@$data['comment']['user_id']) {
            $user_id = $data['comment']['user_id'];
        } elseif (@$data['topic']['user_id']) {
            $user_id = $data['topic']['user_id'];
        }
        if (!Authority()->checkUser('core_shortCode_InvitationCode', $user_id)) {
            return '[' . $s->getName() . '] ' . __('app.Error using short tags');
        }
        $offset = $s->getParameter('offset');
        $limit = $s->getParameter('limit');
        if (!$offset || !$limit) {
            return '[' . $s->getName() . '] ' . __('app.Error using short tags');
        }
        $codes = InvitationCode::query()->limit($limit)->offset($offset - 1)->get();
        return view('App::ShortCode.InvitationCode', ['codes' => $codes]);
    }
    // 轮播
    #[ShortCodeR(name: 'carousel')]
    public function carousel($match, ShortcodeInterface $shortcode, $data)
    {
        // 类型
        $type = $shortcode->getParameter('type');
        // 样式
        $style = $shortcode->getParameter('style');
        // 获取""中的内容
        //        $style = preg_replace('/\"(.*)\"/', '$1', $style);
        // class
        $class = $shortcode->getParameter('class');
        // 获取""中的内容
        //        $class = preg_replace('/\"(.*)\"/', '$1', $class);
        if (!is_numeric($type) || !in_array($type, [1, 2, 3, 4, 5, 6])) {
            return '[' . $shortcode->getName() . '] ' . __('app.Error using short tags');
        }
        // 生成随机id
        $id = Str::random();
        // 获取内容
        $content = $shortcode->getContent();
        // 删除内容中的空格及换行符和html代码
        $content = strip_tags($content);
        $content = preg_replace("/(\\s|\r|\n|\t|\v|\f)+/", '', $content);
        // 把内容转为数组类型
        $content = explode('|', $content);
        // 所有图片
        $images = [];
        foreach ($content as $item) {
            $images[] = json_decode($item);
        }
        return view('App::ShortCode.carousel.' . $type, compact('class', 'style', 'id', 'images'));
    }
    // 视频媒体
    #[ShortCodeR(name: 'media')]
    public function media($match, ShortcodeInterface $shortcode, $data)
    {
        // 网站
        $website = $shortcode->getParameter('website');
        $id = trim(strip_tags($shortcode->getContent()));
        switch ($website) {
            case 'bilibili':
                $website = 'bilibili';
                break;
            case 'youtube':
                $website = 'youtube';
                break;
            default:
                return '[' . $shortcode->getName() . '] ' . __('app.Error using short tags');
        }
        return view('App::ShortCode.media.' . $website, compact('id'));
    }
}