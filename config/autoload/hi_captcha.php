<?php

return [
    'fonts_dir' => BASE_PATH . '/resources/fonts',
    'encryption_driver' => env('CAPTCHA_ENCRYPTION_DRIVER', 'aes'),
    'ttl' => env('CAPTCHA_TTL', 600),
    'characters' => ['1', '2', '3', '4', '6', '7', '8', '9'],
    'a' => [
        'length' => 3,
        'width' => 140,
        'height' => 30,
        'quality' => 90,
        'math' => false,
        'expire' => 120,
        'encrypt' => false,
    ],
    'default' => [
        'length' => 4,
        'width' => 120,
        'height' => 36,
        'quality' => 90,
        'lines' => 6,
        'bgImage' => true,
        'bgColor' => '#ecf2f4',
        'fontColors' => ['#2c3e50', '#c0392b', '#16a085', '#c0392b', '#8e44ad', '#303f9f', '#f57c00', '#795548'],
        'contrast' => -5,
    ],
    'mini' => [
        'length' => 3,
        'width' => 60,
        'height' => 32,
    ],
    'inverse' => [
        'length' => 5,
        'width' => 120,
        'height' => 36,
        'quality' => 90,
        'sensitive' => true,
        'angle' => 12,
        'sharpen' => 10,
        'blur' => 2,
//        'invert' => true,
        'contrast' => -5,
    ]
];
