import { z } from 'zod'

export const MAX_UPLOAD_BYTES = 256 * 1024 * 1024
export const MAX_VECTORS = 1_000_000
export const MAX_DIMENSION = 8192
export const MAX_VALUES = 50_000_000
export const MAX_M = 128
export const MIN_M = 2
export const MAX_EF_CONSTRUCTION = 10_000

export const metricSchema = z.enum(['l2', 'cosine', 'inner_product'])
export const distributionSchema = z.enum(['uniform', 'gaussian', 'clusters'])
export const indexStatusNameSchema = z.enum(['idle', 'building', 'ready', 'error'])

export const healthSchema = z.object({ status: z.string() })

export const apiErrorSchema = z.object({
  error: z.object({ code: z.string(), message: z.string() }),
})

export const datasetListItemSchema = z.object({
  id: z.string(),
  n: z.number().int().nonnegative(),
  dim: z.number().int().positive(),
  createdAt: z.string(),
})

export const datasetListSchema = z.object({
  datasets: z.array(datasetListItemSchema),
})

export const datasetStatsSchema = z.object({
  min: z.array(z.number()),
  max: z.array(z.number()),
  mean: z.array(z.number()),
})

export const datasetSchema = z.object({
  id: z.string(),
  n: z.number().int().nonnegative(),
  dim: z.number().int().positive(),
  preview: z.array(z.array(z.number())),
  stats: datasetStatsSchema.optional(),
  createdAt: z.string().optional(),
  source: z.string().optional(),
  distribution: z.string().optional(),
})

export const createdDatasetSchema = z.object({
  id: z.string(),
  n: z.number().int().positive(),
  dim: z.number().int().positive(),
  preview: z.array(z.array(z.number())),
})

export const generateInputSchema = z
  .object({
    n: z.number().int().min(1, { message: 'n должно быть не меньше 1' }),
    dim: z.number().int().min(1, { message: 'Размерность должна быть не меньше 1' }),
    distribution: distributionSchema,
    seed: z.number().int(),
  })
  .superRefine((value, ctx) => {
    if (value.n > MAX_VECTORS) {
      ctx.addIssue({ code: 'custom', message: `Не больше ${MAX_VECTORS} векторов`, path: ['n'] })
    }
    if (value.dim > MAX_DIMENSION) {
      ctx.addIssue({
        code: 'custom',
        message: `Размерность не больше ${MAX_DIMENSION}`,
        path: ['dim'],
      })
    }
    if (value.n * value.dim > MAX_VALUES) {
      ctx.addIssue({
        code: 'custom',
        message: `n × dim не больше ${MAX_VALUES.toLocaleString('ru-RU')}`,
        path: ['n'],
      })
    }
  })

export const buildIndexInputSchema = z.object({
  datasetId: z.string().min(1, { message: 'Выберите датасет' }),
  M: z
    .number()
    .int({ message: 'M должно быть целым' })
    .min(MIN_M, { message: `M от ${MIN_M} до ${MAX_M}` })
    .max(MAX_M, { message: `M от ${MIN_M} до ${MAX_M}` }),
  efConstruction: z
    .number()
    .int({ message: 'efConstruction должно быть целым' })
    .min(1, { message: `efConstruction от 1 до ${MAX_EF_CONSTRUCTION}` })
    .max(MAX_EF_CONSTRUCTION, {
      message: `efConstruction от 1 до ${MAX_EF_CONSTRUCTION}`,
    }),
  metric: metricSchema,
  heuristic: z.boolean(),
  seed: z.number().int({ message: 'seed должно быть целым' }),
})

export const buildIndexResponseSchema = z.object({
  indexId: z.string(),
  status: z.string(),
})

export const indexStatusSchema = z.object({
  status: indexStatusNameSchema,
  progress: z.number().min(0).max(1),
  elapsedMs: z.number().nonnegative(),
  indexId: z.string(),
  error: z.string().nullable(),
})

export const degreeBucketSchema = z.object({
  degree: z.number().int().nonnegative(),
  count: z.number().int().nonnegative(),
})

export const degreeDistributionSchema = z.object({
  min: z.number().int().nonnegative(),
  max: z.number().int().nonnegative(),
  mean: z.number().nonnegative(),
  p50: z.number().int().nonnegative(),
})

export const indexStatsSchema = z.object({
  nodes: z.number().int().nonnegative(),
  maxLevel: z.number().int().nonnegative(),
  entryPoint: z.number().int().nonnegative(),
  nodesPerLevel: z.array(z.number().int().nonnegative()),
  degreeDistribution: degreeDistributionSchema,
  degreeHistogram: z.array(degreeBucketSchema),
  metric: metricSchema,
  M: z.number().int().positive(),
  efConstruction: z.number().int().positive(),
  indexId: z.string().optional(),
  datasetId: z.string().optional(),
})

export const indexProgressEventSchema = z.object({
  progress: z.number().min(0).max(1),
  processed: z.number().int().nonnegative(),
  total: z.number().int().nonnegative(),
})

export const indexDoneEventSchema = z.object({ indexId: z.string() })

export const indexErrorEventSchema = z.object({ message: z.string() })

export type DatasetListItem = z.infer<typeof datasetListItemSchema>
export type Dataset = z.infer<typeof datasetSchema>
export type DatasetStats = z.infer<typeof datasetStatsSchema>
export type CreatedDataset = z.infer<typeof createdDatasetSchema>
export type GenerateInput = z.infer<typeof generateInputSchema>
export type BuildIndexInput = z.infer<typeof buildIndexInputSchema>
export type Metric = z.infer<typeof metricSchema>
export type Distribution = z.infer<typeof distributionSchema>
export type IndexStatusName = z.infer<typeof indexStatusNameSchema>
export type IndexStatus = z.infer<typeof indexStatusSchema>
export type IndexStats = z.infer<typeof indexStatsSchema>
export type DegreeBucket = z.infer<typeof degreeBucketSchema>
export type IndexProgressEvent = z.infer<typeof indexProgressEventSchema>
export type IndexDoneEvent = z.infer<typeof indexDoneEventSchema>
export type IndexErrorEvent = z.infer<typeof indexErrorEventSchema>
