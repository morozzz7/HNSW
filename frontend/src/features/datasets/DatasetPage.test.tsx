import { fireEvent, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { renderWithProviders } from '@/test/render'
import { DatasetPage } from './DatasetPage'

describe('DatasetPage', () => {
  it('lists the seeded datasets and shows the first preview', async () => {
    renderWithProviders(<DatasetPage />)

    expect(await screen.findByText('ds_000001')).toBeInTheDocument()
    expect(screen.getByText('ds_000002')).toBeInTheDocument()
    expect(await screen.findByText('Превью')).toBeInTheDocument()
    expect(screen.getByText('Статистика координат')).toBeInTheDocument()
    expect(await screen.findByText('Датасет ds_000001')).toBeInTheDocument()
  })

  it('generates a dataset and selects it', async () => {
    const user = userEvent.setup()
    renderWithProviders(<DatasetPage />)

    await screen.findByText('ds_000001')

    const vectors = screen.getByLabelText('Векторов, n')
    await user.clear(vectors)
    await user.type(vectors, '120')

    const dim = screen.getByLabelText('Размерность, dim')
    await user.clear(dim)
    await user.type(dim, '8')

    await user.click(screen.getByRole('button', { name: /Сгенерировать/ }))

    expect(await screen.findByText('ds_000003')).toBeInTheDocument()
    expect(await screen.findByText('Датасет ds_000003')).toBeInTheDocument()
    expect(screen.getByText('n × dim = 960 компонентов, использовано 0%.')).toBeInTheDocument()
  })

  it('rejects a file with an unsupported extension before uploading', async () => {
    renderWithProviders(<DatasetPage />)

    await screen.findByText('ds_000001')

    const input = document.querySelector('input[type="file"]')
    expect(input).not.toBeNull()
    const file = new File(['nope'], 'notes.txt', { type: 'text/plain' })
    fireEvent.change(input as HTMLInputElement, { target: { files: [file] } })

    expect(await screen.findByText('Допустимы только файлы .csv и .json')).toBeInTheDocument()
  })
})
