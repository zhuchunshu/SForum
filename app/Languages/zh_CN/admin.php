<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
return [
    'setting' => [
        'language' => '站点默认语言',
        'Whether to enable rendering of twemoji' => '是否开启渲染twemoji',
        'twemoji static resource library' => 'twemoji静态资源库',
        'twemoji image width' => 'twemoji图片宽度',
        'twemoji image height' => 'twemoji图片高度',
        'Whether to enable rendering of owo expressions' => '是否开启渲染owo表情',
    ],
    'wealth' => [
        'exchange rate' => '兑换比例',
        'money unit name' => ':name 单位名',
        'money name' => '余额名称',
        'credit name' => '积分名称',
        'golds name' => '金币名称',
        'exp name' => '经验名称',
        'how many' => ':number :first 等于多少 :last',
        'close redemption' => '关闭 :first 兑换 :last',
        'exchange alert' => '兑换比例保存后不建议修改，可能会造成财富数据错乱！！！ 兑换比例不建议过高 ，建议1-100',
    ],
    'turn on' => '开启',
    'turn off' => '关闭',
    'default' => '默认: :default',
    'current' => '当前: :current',
    'user' => [
        'pm' => [
            'msg maxlength' => '消息最大长度',
            'msg reserve' => '消息保留时长',
        ],
    ],
    'is reserved forever' => ':reserve 为永久保留',
    'Unit:day' => '单位:天',
];
