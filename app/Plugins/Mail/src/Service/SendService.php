<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Mail\src\Service;

use Hyperf\Di\Annotation\AnnotationCollector;
class SendService
{
    // 获取所有发信服务接口
    public function get_services() : array
    {
        $services = [];
        foreach (AnnotationCollector::getClassesByAnnotation(\App\Plugins\Mail\src\Annotation\SendService::class) as $key => $value) {
            if (new $key() instanceof \App\Plugins\Mail\src\SendServiceInterface) {
                $services[md5($key)] = ['name' => (new $key())->get_service_name(), 'handler' => (new $key())->handler(), 'view' => (new $key())->view(), 'class' => $key];
            }
        }
        return $services;
    }
    /**
     * 获取所有发信服务接口.
     */
    public function get_handlers() : array
    {
        $handlers = [];
        foreach (AnnotationCollector::getClassesByAnnotation(\App\Plugins\Mail\src\Annotation\SendService::class) as $key => $value) {
            if (new $key() instanceof \App\Plugins\Mail\src\SendServiceInterface) {
                $handlers[] = (new $key())->handler();
            }
        }
        return $handlers;
    }
    /**
     * 发送邮件.
     * @param $email
     * @param $subject
     * @param $body
     * @return mixed
     */
    public function send($email, $subject, $body)
    {
        $service = get_options('mail_service', 'ca583971fcbccdf5d7ed77cf4c471ac5');
        $class = $this->get_services()[$service]['class'];
        $content = make_template(plugin_path('Mail/resources/template/sforum.htm'), ['web_name' => get_options('web_name', config('app_name')), 'title' => $subject, 'content' => $body, 'unlink' => url('/user/setting'), 'url' => url(), 'app_name' => get_options('APP_NAME', config('app_name')), 'date' => date('Y')]);
        $body = $content;
        return (new $class())->send($email, $subject, $body);
    }
    /**
     * 异步发送邮件.
     * @param $email
     * @param $subject
     * @param $body
     */
    public function async_send($email, $subject, $body)
    {
        go(function () use($email, $subject, $body) {
            $this->send($email, $subject, $body);
        });
    }
}