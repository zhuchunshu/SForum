<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/super-forum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/super-forum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use App\Plugins\Topic\src\Models\Topic;
use Carbon\Carbon;

/**
 * @property int $id
 * @property string $user_id
 * @property string $path
 * @property string $url
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UserUpload extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'user_upload';

    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['user_id', 'created_at', 'updated_at', 'path', 'url', 'post_id'];

    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];

    public function user()
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }

    public function topic()
    {
        return $this->belongsTo(Topic::class, 'topic_id', 'id');
    }
}
