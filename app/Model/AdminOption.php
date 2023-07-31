<?php

declare (strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace App\Model;

use Hyperf\ModelCache\Cacheable;
use Hyperf\ModelCache\CacheableInterface;
class AdminOption extends Model implements CacheableInterface
{
    use Cacheable;
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'admin_options';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'name', 'value', 'created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer'];
}