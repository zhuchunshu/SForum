<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
/**
 * @property int $id
 * @property string $name
 * @property string $link
 * @property string $icon
 * @property int $to_sort
 * @property int $_blank
 * @property string $description
 */
class FriendLink extends Model
{
    public bool $timestamps = false;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'friend_links';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'name', 'link', 'to_sort', 'icon', 'hidden', 'target', 'description'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'to_sort' => 'integer', '_blank' => 'integer', 'hidden' => 'integer'];
}