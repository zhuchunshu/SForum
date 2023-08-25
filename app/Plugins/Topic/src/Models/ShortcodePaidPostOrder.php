<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Topic\src\Models;

use App\Model\Model;

/**
 * @property int $id
 * @property int $post_id
 * @property int $user_id
 * @property \Carbon\Carbon $created_at
 * @property \Carbon\Carbon $updated_at
 */
class ShortcodePaidPostOrder extends Model
{
    /**
     * The table associated with the model.
     */
    protected ?string $table = 'shortcode_paid_post_order';

    /**
     * The attributes that are mass assignable.
     */
    protected array $fillable = ["id","post_id","user_id","created_at","updated_at"];

    /**
     * The attributes that should be cast to native types.
     */
    protected array $casts = ['id' => 'integer', 'post_id' => 'integer', 'user_id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}
