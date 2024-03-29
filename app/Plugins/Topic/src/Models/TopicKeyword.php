<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
use Carbon\Carbon;
/**
 * @property int $id
 * @property string $name
 * @property string $user_id
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class TopicKeyword extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'topic_keywords';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'name', 'created_at', 'updated_at', 'user_id'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
    public function kw()
    {
        return $this->hasMany(TopicKeywordsWith::class, 'with_id', 'id');
    }
    public function user()
    {
        return $this->belongsTo(User::class, 'user_id', 'id');
    }
}